//  Crypto-Obscured Forwarder
//
//  Copyright (C) 2018 Rui NI <ranqus@gmail.com>
//
//  This file is part of Crypto-Obscured Forwarder.
//
//  Crypto-Obscured Forwarder is free software: you can redistribute it
//  and/or modify it under the terms of the GNU General Public License
//  as published by the Free Software Foundation, either version 3 of
//  the License, or (at your option) any later version.
//
//  Crypto-Obscured Forwarder is distributed in the hope that it will be
//  useful, but WITHOUT ANY WARRANTY; without even the implied warranty
//  of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with Crypto-Obscured Forwarder. If not, see
//  <http://www.gnu.org/licenses/>.

package application

import (
	"bufio"
	"bytes"
	"fmt"
	golog "log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/reinit/coward/common/config"
	"github.com/reinit/coward/common/logger"
	"github.com/reinit/coward/common/parameter"
	"github.com/reinit/coward/common/print"
	"github.com/reinit/coward/common/role"
	"github.com/reinit/coward/common/rw"
)

// Application represents the COWARD application
type Application interface {
	Help() error
	Version() string
	ExecuteArgumentInput(parameters []string) error
	ExecuteParameter(
		execCfg ExecuteConfig, name string, parameter string) error
	ExecuteConfiguration(
		execCfg ExecuteConfig, name string, config interface{}) error
}

// application implements Application
type application struct {
	intro     []byte
	introTail []byte
	about     []byte
	exeName   string
	version   string
	roles     role.Roler
	logger    logger.Logger
	printer   print.Print
}

const roleListFormat = "    %%%ds    %%s\r\n"

// New build a new COWARD application according to Config
func New(log logger.Logger, cfg Config) Application {
	var exeName string
	var intro string

	about := aboutBanner + "\r\n"

	if cfg.Banner != "" {
		intro = cfg.Banner
	} else if cfg.Name != "" {
		intro = NamedBanner
	} else {
		intro = DefaultBanner
	}

	if cfg.Name == "" {
		cfg.Name = Name
	} else {
		about += aboutPoweredByBanner + "\r\n"
	}

	if cfg.Version == "" {
		cfg.Version = version
	}

	if cfg.URL == "" {
		cfg.URL = URL
	}

	if cfg.Copyright == "" {
		cfg.Copyright = Copyright
	}

	intro = strings.Replace(intro, "<Name>", cfg.Name, -1)
	intro = strings.Replace(intro, "<Version>", cfg.Version, -1)
	intro = strings.Replace(intro, "<URL>", cfg.URL, -1)
	intro = strings.Replace(intro, "<Copyright>", cfg.Copyright, -1)

	about = strings.Replace(about, "<Name>", cfg.Name, -1)
	about = strings.Replace(about, "<Version>", cfg.Version, -1)
	about = strings.Replace(about, "<URL>", cfg.URL, -1)
	about = strings.Replace(about, "<Copyright>", cfg.Copyright, -1)
	about = strings.Replace(about, "<COWARD:Name>", Name, -1)
	about = strings.Replace(about, "<COWARD:Version>", version, -1)

	registeredComponents := make(role.Components, 0, 16)
	registeredRoles := make(role.Registrations, 0, 16)

	for _, r := range cfg.Components {
		switch reg := r.(type) {
		case func() role.Registration:
			registeredRoles = append(registeredRoles, reg())

		default:
			registeredComponents = append(registeredComponents, reg)
		}
	}

	rol, rolErr := role.NewRoler(role.Config{
		Roles:      registeredRoles,
		Components: registeredComponents,
		OnListScreen: func(
			w print.Common, maxRoleNameLen int, roles role.Roles) {
			format := fmt.Sprintf(roleListFormat, maxRoleNameLen)

			if len(roles) <= 0 {
				w.Writeln([]byte(fmt.Sprintln(
					"Sorry, there is no any available role "+
						"registered for this application.")),
					1, 2, 1)

				return
			}

			w.Writeln([]byte(fmt.Sprintln(
				"Please select one of following roles to continue:")),
				1, 2, 1)

			for key, val := range roles {
				w.Writeln([]byte(fmt.Sprintf(format, key, val.Description)),
					0, maxRoleNameLen+8, 1)
			}
		},
		OnUndefined: func(
			w print.Common,
			name string,
			maxRoleNameLen int,
			roles role.Roles,
		) {
			format := fmt.Sprintf(roleListFormat, maxRoleNameLen)

			if name == "" {
				w.Writeln([]byte(fmt.Sprintln(
					"You must specify a role for COWARD:")), 1, 2, 1)
			} else {
				w.Writeln([]byte(fmt.Sprintf("\"%s\" is not a registered "+
					"role. You could only use roles from below:\r\n", name)),
					1, 2, 1)
			}

			for key, val := range roles {
				w.Writeln([]byte(fmt.Sprintf(format, key, val.Description)),
					0, maxRoleNameLen+8, 1)
			}
		},
	})

	if rolErr != nil {
		panic("Can't register roles due to error: " + rolErr.Error())
	}

	if len(os.Args) > 0 {
		exeName = filepath.Base(os.Args[0])
	}

	if exeName == "" || exeName == "." {
		exeName = "coward"
	}

	cw := &application{
		intro:     []byte(intro),
		introTail: []byte("\r\n"),
		about:     []byte(about + "\r\n"),
		exeName:   exeName,
		version:   cfg.Version,
		roles:     rol,
		logger:    log,
		printer: print.New(print.Config{
			MaxLineWidth: func() int {
				return 80
			},
		}),
	}

	return cw
}

func (c *application) Help() error {
	screen := bytes.NewBuffer(make([]byte, 0, 256))
	printer := c.printer.Buffer(c.intro, c.introTail)

	printer.Writeln([]byte(fmt.Sprintf(helpUsage, c.exeName)), 1, 4, 1)
	printer.Write([]byte("\r\n"))

	printer.Writeln([]byte("Execute options are:\r\n"), 1, 1, 1)

	helpItemSpaceLen := c.roles.MaxRoleNameLen() - 3

	if helpItemSpaceLen < 2 {
		helpItemSpaceLen = 2
	}

	if c.logger == nil {
		printer.Writeln([]byte(fmt.Sprintf(
			helpUsageSlient,
			strings.Repeat(" ", helpItemSpaceLen))), 4, helpItemSpaceLen+11, 1)
		printer.Writeln([]byte(fmt.Sprintf(
			helpUsageDebug,
			strings.Repeat(" ", helpItemSpaceLen))), 4, helpItemSpaceLen+11, 1)
		printer.Writeln([]byte(fmt.Sprintf(
			helpUsageLog,
			strings.Repeat(" ", helpItemSpaceLen))), 4, helpItemSpaceLen+11, 1)
	}

	printer.Writeln([]byte(fmt.Sprintf(
		helpUsageDaemon,
		strings.Repeat(" ", helpItemSpaceLen))), 4, helpItemSpaceLen+11, 1)
	printer.Writeln([]byte(fmt.Sprintf(
		helpUsageParam,
		strings.Repeat(" ", helpItemSpaceLen))), 4, helpItemSpaceLen+11, 1)

	printer.Write([]byte("\r\n\r\n"))

	c.roles.List(printer)

	printer.WriteTo(screen)

	screen.WriteTo(os.Stdout)

	return nil
}

func (c *application) Version() string {
	return c.version
}

func (c *application) loadConfigurationFromFile(
	path string) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	file, fileErr := os.OpenFile(path, os.O_RDONLY, 0)

	if fileErr != nil {
		return nil, ErrFailedToOpenParameterFile
	}

	defer file.Close()

	buf.ReadFrom(file)

	data := bytes.TrimSpace(buf.Bytes())

	if len(data) <= 0 {
		return nil, ErrParameterFileEmpty
	}

	return data, nil
}

func (c *application) buildRunConfigFromParam(
	parameters []string) (ExecuteConfig, int, error) {
	breakLoop := false
	result := ExecuteConfig{
		Daemom:    false,
		Slient:    false,
		Debug:     false,
		LogFile:   "",
		ParamFile: "",
		Shutdown:  nil,
		Booted:    nil,
	}
	lastIdx := 0
	paramLen := len(parameters)
	paramLastIdx := paramLen - 1

	for {
		if lastIdx > paramLastIdx {
			return ExecuteConfig{}, 0, ErrExecuteOptionEndedBeforeRoleName
		}

		trimedParam := strings.ToLower(strings.TrimSpace(parameters[lastIdx]))

		if trimedParam == "" {
			lastIdx++

			continue
		}

		switch {
		case c.logger == nil && trimedParam == "-slient":
			fallthrough
		case c.logger == nil && trimedParam == "-s":
			result.Slient = true

		case c.logger == nil && trimedParam == "-debug":
			result.Debug = true

		case c.logger == nil && trimedParam == "-log":
			fallthrough
		case c.logger == nil && trimedParam == "-l":
			if lastIdx+1 >= paramLen {
				return ExecuteConfig{}, 0, ErrLogFileMustBeSpecified
			}

			lastIdx++

			result.LogFile = strings.TrimSpace(parameters[lastIdx])

			if result.LogFile == "" {
				return ExecuteConfig{}, 0, ErrLogFileMustBeSpecified
			}

		case c.logger == nil && trimedParam == "-daemon":
			fallthrough
		case c.logger == nil && trimedParam == "-d":
			result.Daemom = true

		case trimedParam == "-param":
			fallthrough
		case trimedParam == "-p":
			if lastIdx+1 >= paramLen {
				return ExecuteConfig{}, 0, ErrConfigFileMustBeSpecified
			}

			lastIdx++

			result.ParamFile = strings.TrimSpace(parameters[lastIdx])

			if result.ParamFile == "" {
				return ExecuteConfig{}, 0, ErrConfigFileMustBeSpecified
			}

		default:
			if trimedParam[0] == '-' {
				return ExecuteConfig{}, 0, ErrUnknownExecuteOption
			}

			breakLoop = true
		}

		if breakLoop {
			break
		}

		lastIdx++
	}

	return result, lastIdx, nil
}

func (c *application) escapeArgumentString(
	parameters []string, start int) []byte {
	paramString := []byte("")

	for _, param := range parameters[start:] {
		trimedParam := bytes.TrimSpace([]byte(param))

		if len(trimedParam) <= 0 {
			continue
		}

		if trimedParam[0] == '-' {
			paramString = append(paramString, trimedParam...)
			paramString = append(paramString, ' ')

			continue
		}

		if bytes.Contains(trimedParam, []byte(" ")) {
			paramString = append(paramString, '"')
			paramString = append(paramString, bytes.Replace(
				bytes.Replace(trimedParam, []byte("\\"), []byte("\\\\"), -1),
				[]byte("\""), []byte("\\\""), -1)...)
			paramString = append(paramString, []byte("\" ")...)

			continue
		}

		paramString = append(paramString, trimedParam...)
		paramString = append(paramString, ' ')
	}

	return bytes.TrimSpace(paramString)
}

func (c *application) run(
	init func() (ExecuteConfig, int, error),
	exec func(cfg ExecuteConfig, roleParamStart int, p print.Printer) error,
) error {
	var err error

	printer := c.printer.Printer(os.Stdout)

	defer func() {
		if err != nil {
			sampleOut := ""

			switch e := err.(type) {
			case *config.ParseError:
				// 10 = 7 + 1 + \r\n
				sampleOut = e.Sample(printer.MaxLen() - 10)

			case *parameter.SyntaxError:
				// 10 = 7 + 1 + \r\n
				sampleOut = e.Sample(printer.MaxLen() - 10)
			}

			if sampleOut != "" {
				sampleOut = "\r\n\r\n" + sampleOut
			}

			printer.Writeln(
				[]byte(fmt.Sprintf(
					"<ERR> Command has failed due to error: %s%s",
					err, sampleOut)),
				1, 7, 1)
		} else {
			printer.Writeln(
				[]byte(fmt.Sprintf("<OK>  Command successfully completed")),
				1, 7, 1)
		}

		printer.Write(c.introTail)
		printer.Close()
	}()

	cfg, roleStartPos, initErr := init()

	if initErr != nil {
		err = initErr

		printer.Write(c.about)

		return err
	}

	if cfg.Slient {
		printer.Close()
	}

	printer.Write(c.about)

	err = exec(cfg, roleStartPos, printer)

	return err
}

func (c *application) ExecuteArgumentInput(parameters []string) error {
	return c.run(func() (ExecuteConfig, int, error) {
		exeCfg, roleParamStart, exeCfgErr := c.buildRunConfigFromParam(
			parameters)

		if exeCfgErr != nil {
			return exeCfg, roleParamStart, exeCfgErr
		}

		return exeCfg, roleParamStart, nil
	}, func(
		execCfg ExecuteConfig,
		roleParamStart int,
		printer print.Printer,
	) error {
		cmdParam := c.escapeArgumentString(parameters, roleParamStart+1)

		return c.execute(
			printer,
			func(log logger.Logger) (role.Role, error) {
				var param []byte
				var err error

				if execCfg.ParamFile != "" {
					param, err = c.loadConfigurationFromFile(
						execCfg.ParamFile)

					if err != nil {
						return nil, err
					}

					param = append(param, []byte("\r\n")...)
				}

				param = append(param, cmdParam...)

				return c.roles.InitParameterString(
					printer,
					parameters[roleParamStart],
					param,
					log,
				)
			},
			execCfg,
		)
	})
}

func (c *application) ExecuteParameter(
	exeCfg ExecuteConfig, name string, parameter string) error {
	return c.run(func() (ExecuteConfig, int, error) {
		return exeCfg, 0, nil
	}, func(
		execCfg ExecuteConfig,
		roleParamStart int,
		printer print.Printer,
	) error {
		return c.execute(
			printer,
			func(log logger.Logger) (role.Role, error) {
				return c.roles.InitParameterString(
					printer, name, []byte(parameter), log)
			},
			execCfg,
		)
	})
}

func (c *application) ExecuteConfiguration(
	exeCfg ExecuteConfig, name string, config interface{}) error {
	return c.run(func() (ExecuteConfig, int, error) {
		return exeCfg, 0, nil
	}, func(
		execCfg ExecuteConfig,
		roleParamStart int,
		printer print.Printer,
	) error {
		return c.execute(
			printer,
			func(log logger.Logger) (role.Role, error) {
				return c.roles.Init(printer, name, config, log)
			},
			execCfg,
		)
	})
}

func (c *application) execute(
	printer print.Printer,
	roleGen func(log logger.Logger) (role.Role, error),
	config ExecuteConfig,
) error {
	var log logger.Logger

	// Buffer 1 for close notify because the shutdown function
	// or `Unspawn` will try to write it. But since no one is
	// reading it during that time, it will block and never returned.
	// So it's necessary to buffer it so the method will return and we
	// can read it afterwards
	closedNotify := make(role.UnspawnNotifier, 1)
	signals := make(chan os.Signal)
	breakLoop := false

	if c.logger == nil {
		switch strings.ToLower(config.LogFile) {
		case "":
			if config.Slient {
				log = logger.NewDitch()
			} else if config.Debug {
				log = logger.NewScreen(printer)
			} else {
				log = logger.NewScreenNonDebug(printer)
			}

		case "stdout":
			if config.Debug {
				log = logger.NewWrite(rw.NewMutexedWriter(os.Stdout))
			} else {
				log = logger.NewWriteNonDebug(rw.NewMutexedWriter(os.Stdout))
			}

		case "stderr":
			if config.Debug {
				log = logger.NewWrite(rw.NewMutexedWriter(os.Stderr))
			} else {
				log = logger.NewWriteNonDebug(rw.NewMutexedWriter(os.Stderr))
			}

		default:
			file, fileErr := os.Create(config.LogFile)

			if fileErr != nil {
				return fileErr
			}

			defer file.Close()

			bFile := bufio.NewWriter(file)

			defer bFile.Flush()

			if config.Debug {
				log = logger.NewWrite(rw.NewMutexedWriter(bFile))
			} else {
				log = logger.NewWriteNonDebug(rw.NewMutexedWriter(bFile))
			}
		}
	} else {
		log = c.logger
	}

	golog.SetOutput(log)

	defer golog.SetOutput(os.Stdout)

	// If we can manually shutdown the application through the Shutdown
	// channel, then there will be no need for monitering os signals as
	// the Shutdown channel is designed for integration
	if config.Shutdown == nil {
		signal.Notify(
			signals, syscall.SIGINT,
			syscall.SIGTERM, syscall.SIGHUP)

		defer signal.Stop(signals)
	}

	for {
		if breakLoop {
			break
		}

		r, rErr := roleGen(log)

		if rErr != nil {
			return rErr
		}

		spawnErr := r.Spawn(closedNotify)

		if spawnErr != nil {
			r.Unspawn()

			<-closedNotify

			return spawnErr
		}

		select {
		case config.Booted <- true:
		default:
		}

		select {
		case sig := <-signals:
			switch sig {
			case syscall.SIGINT:
				fmt.Print("\r")
				fallthrough

			case syscall.SIGTERM:
				breakLoop = true
				fallthrough

			case syscall.SIGHUP:
				if !config.Daemom {
					breakLoop = true
				}

				unspawnErr := r.Unspawn()

				if unspawnErr != nil {
					return unspawnErr
				}

				<-closedNotify
			}

		case <-closedNotify:
			breakLoop = true

			unspawnErr := r.Unspawn()

			if unspawnErr != nil {
				return unspawnErr
			}

			<-closedNotify

		case breakLoop = <-config.Shutdown:
			// When shutdown channel send true, we shutdown
			// the application, otherwise the application will
			// be just reloaded
			unspawnErr := r.Unspawn()

			if unspawnErr != nil {
				return unspawnErr
			}

			<-closedNotify
		}
	}

	return nil
}
